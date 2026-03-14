<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://www.example.com/local-functions">
<?spec xslt#shallow-copy?>
<!-- xsl:copy in a function: select attribute sets a singleton focus  -->                
<xsl:template match="/">
    <out>
      <xsl:sequence select="for $x in * return f:copy-node($x)"/>
    </out>
</xsl:template>

<xsl:function name="f:copy-node">
    <xsl:param name="n" as="node()"/>
    <xsl:copy select="$n">
        <xsl:attribute name="position" select="position()"/>
        <xsl:attribute name="last" select="last()"/>
        <xsl:sequence select="*!f:copy-node(.)"/>
    </xsl:copy>
</xsl:function>
     
</xsl:stylesheet>