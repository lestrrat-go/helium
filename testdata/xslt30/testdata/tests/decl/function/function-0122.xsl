<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- function using EQName must be in a namespace-->

  <xsl:function name="Q{}count-elements">
    <xsl:param name="doc"/>
    <xsl:sequence select="count($doc//*)"/>
  </xsl:function>

  <xsl:template match="/">
    <out>
      <xsl:value-of select="23"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
