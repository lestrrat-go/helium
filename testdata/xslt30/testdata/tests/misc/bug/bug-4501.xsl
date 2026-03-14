<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#named-templates?>
<!-- From Jeni Tennison, 12 Jan 2001
I have a weird problem with recursion in Saxon 6.1.  In the following
stylesheet, if there's nothing after the recursive call to the named
template, then the elements that are produced are not nested, as they
should be.  If I add <xsl:comment>end match</xsl:comment> after the
recursive call it works fine.
-->


<xsl:key name="leaf-elements" match="*[not(*)]" use="name()" />

<xsl:template match="/">
   <xsl:apply-templates select="//*[not(*) and count(.|key('leaf-elements', name())[1]) = 1]"/>
</xsl:template>

<xsl:template match="*">
   <xsl:variable name="node" select="." />
   <xsl:variable name="matches" select="key('leaf-elements', name())[position() != 1]" />
   <xsl:for-each select="$matches">
      <xsl:call-template name="compare">
         <xsl:with-param name="node1" select="$node" />
         <xsl:with-param name="node2" select="." />
      </xsl:call-template>
   </xsl:for-each>
</xsl:template>

<xsl:template name="compare">
   <xsl:param name="node1" />
   <xsl:param name="node2" />
   <match>
      <!-- content removed for brevity -->
      <xsl:if test="count($node1/parent::* | $node2/parent::*) = 2 and
                    name($node1/parent::*) = name($node2/parent::*)">
         <xsl:call-template name="compare">
            <xsl:with-param name="node1" select="$node1/parent::*" />
            <xsl:with-param name="node2" select="$node2/parent::*" />
         </xsl:call-template>
      </xsl:if>
   </match>
</xsl:template>

</xsl:stylesheet>
