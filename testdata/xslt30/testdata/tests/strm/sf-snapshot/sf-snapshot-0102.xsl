<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 xmlns:f="http://example.com/functions" 
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 exclude-result-prefixes="f xs">
 
<!-- Test that the results of the fn:snapshot function are equivalent to the 
 implementation given in the XSLT spec --> 
 
<xsl:template name="main">
 <xsl:source-document streamable="yes" href="sf-snapshot-01.xml">
  <out>
   <xsl:for-each select="snapshot(//*)">
    <xsl:variable name="p" select="position()" as="xs:integer"/>
    <xsl:variable name="s" select="."/>
    <xsl:variable name="t" select="f:snapshot((doc('sf-snapshot-01.xml')//*)[$p])"/>
    <xsl:if test="not(deep-equal($s, $t) and deep-equal($s/root(), $t/root()))"> 
      <wrong position="{$p}" node="{generate-id(.)}" path="{string-join(ancestor-or-self::*/name(), '/')}">
        <actual name="{name($s)}"><xsl:sequence select="$s/root()"/></actual>
        <expected name="{name($t)}"><xsl:sequence select="$t/root()"/></expected>
      </wrong>
    </xsl:if>
   </xsl:for-each>
  </out>
 </xsl:source-document>   
</xsl:template> 


 


<!-- Following functions are the non-streamable implementation of fn:snapshot defined in the XSLT 3.0 spec -->

<xsl:function name="f:snapshot" as="node()?" new-each-time="maybe">
  <xsl:param name="origin" as="node()?"/>
  
  <!-- create a copy of the tree containing the supplied node, retaining only
       * the supplied node with its attributes and namespaces
       * the ancestors of the supplied node with their attributes and namespaces
       * the descendants of the supplied node with their attributes and namespaces
       -->
  <xsl:variable name="root-copy" as="node()">
    <xsl:apply-templates select="root($origin)" mode="snapshot">
      <xsl:with-param name="origin" select="$origin" tunnel="yes"/>
    </xsl:apply-templates>
  </xsl:variable>
  
  <!-- find and return the node in the copied tree that corresponds to the origin node
       --> 
  <xsl:sequence select="$root-copy/descendant-or-self::node()/(.|@*|namespace::*)
      [f:corresponds(., $origin)]"/>
</xsl:function>

<xsl:template match="." mode="snapshot">
  <xsl:param name="origin" as="node()" tunnel="yes"/>
  <xsl:choose>
    <xsl:when test=". is $origin">
      <xsl:copy-of select="." validation="preserve"/>
    </xsl:when>
    <xsl:when test=". intersect $origin/ancestor::node()">
      <xsl:copy validation="strip">
        <xsl:copy-of select="@*" validation="preserve"/>
        <xsl:apply-templates mode="snapshot"/>
      </xsl:copy>
    </xsl:when>
    <xsl:otherwise/>
  </xsl:choose>
</xsl:template>

<!-- f:corresponds compares two nodes and returns true if all the following are true:
     * they are at the same depth in their respective trees
     * they have the same name (or none)
     * they have the same node kind
     The tests on node name and node kind are needed only when the origin node is an
     attribute or namespace.
     -->
<xsl:function name="f:corresponds" as="xs:boolean">
  <xsl:param name="node1" as="node()"/>
  <xsl:param name="node2" as="node()"/>
  <xsl:sequence select="
    count($node1/ancestor::node()) = count($node2/ancestor::node()) and
    deep-equal(node-name($node1), node-name($node2)) and
    f:node-kind($node1) = f:node-kind($node2)"/>    
</xsl:function>

<!-- f:node-kind returns a string identifying the node kind of the supplied node
     -->
<xsl:function name="f:node-kind" as="xs:string">
  <xsl:param name="node" as="node()"/>
  <xsl:choose>
    <xsl:when test="$node instance of document-node()">D</xsl:when>
    <xsl:when test="$node instance of element(*)">E</xsl:when>
    <xsl:when test="$node instance of attribute(*)">A</xsl:when>
    <xsl:when test="$node instance of text()">T</xsl:when>
    <xsl:when test="$node instance of comment()">C</xsl:when>
    <xsl:when test="$node instance of processing-instruction()">P</xsl:when>
    <xsl:when test="$node instance of namespace-node()">N</xsl:when>
  </xsl:choose>
</xsl:function>



</xsl:stylesheet>