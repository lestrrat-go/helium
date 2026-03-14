<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:app="http://app.com/" exclude-result-prefixes="app">

  <!-- simple xsl:function, @required='yes' is allowed in 3.0 but not in 2.0 -->

  <xsl:function name="app:count-elements">
    <xsl:param name="doc" required='yes'/>
    <xsl:sequence select="exists($doc//*)"/>   
  </xsl:function>

  <xsl:template match="/">
    <out>
      <xsl:value-of select="app:count-elements(/)"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
