<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://internal.snapshot.com/" 
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 exclude-result-prefixes="f xs"
  expand-text="yes">
 
<!-- Test that the results of the fn:snapshot function are equivalent to the 
 revised implementation given in the XSLT spec -->
 
<xsl:template match="/">
 <out>
  <xsl:for-each select="descendant-or-self::node()/(. | @* | namespace::* ) ">
    <xsl:if test="not(deep-equal(snapshot(.), f:snapshot(.)) 
                       and deep-equal(snapshot(.)/root(), f:snapshot(.)/root()))"> 
      <wrong path="{path(.)}" node-ok="{deep-equal(snapshot(.), f:snapshot(.))}" root-ok="{deep-equal(snapshot(.)/root(), f:snapshot(.)/root())}">
        <expected is-doc="{snapshot(.)/root() instance of document-node()}">
          <xsl:sequence select="snapshot(.)/root() treat as document-node()"/>
        </expected>
        <actual is-doc="{f:snapshot(.)/root() instance of document-node()}">
          <xsl:sequence select="f:snapshot(.)/root() treat as document-node()"/>
        </actual>
      </wrong>
    </xsl:if>
  </xsl:for-each>
 </out>
</xsl:template> 
 
    
<xsl:include href="snapshot-equivalent.xsl"/>
 

</xsl:stylesheet>