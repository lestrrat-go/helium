<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:my="http://my.uri/">

<?error XTDE0070?>



  <xsl:template name="main">
    <out>
      <xsl:sequence select="my:f(3)"/>
    </out>
  </xsl:template>
  
  <xsl:function name="my:f">
    <xsl:param name="n"/>
    <xsl:sequence select="."/>
  </xsl:function>




</xsl:stylesheet>