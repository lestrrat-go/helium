<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- function declaration and invocation both use EQName -->

  <xsl:function name="Q{http://app.com/}count-elements">
    <xsl:param name="doc"/>
    <xsl:sequence select="count($doc//*)"/>
  </xsl:function>

  <xsl:template match="/">
    <out>
      <xsl:value-of select="Q{http://app.com/}count-elements(/)"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
