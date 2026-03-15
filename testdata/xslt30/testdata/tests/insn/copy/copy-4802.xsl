<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://www.example.com/local-functions">
<?spec xslt#shallow-copy?>
<!-- Test for simple identity transformation: xsl:copy in a function  -->                
<xsl:template match="/">
    <out>
      <xsl:sequence select="f:copy-node(.)"/>
    </out>
</xsl:template>

<xsl:function name="f:copy-node" as="node()">
  <xsl:param name="n" as="node()"/>
  <xsl:copy select="$n">
    <xsl:for-each select="$n/(@*, node())">
        <xsl:sequence select="f:copy-node(.)"/>
    </xsl:for-each>
  </xsl:copy>
</xsl:function>
     
</xsl:stylesheet>