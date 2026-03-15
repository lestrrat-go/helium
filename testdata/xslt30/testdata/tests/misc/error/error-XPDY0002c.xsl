<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:app="http://app.com/">

<!-- error: context item is not set on entry to a stylesheet function -->
<?spec xslt#errors?><?error XPDY0002?>

<xsl:function name="app:count-elements">
    <xsl:sequence select="count(//*)"/>   
</xsl:function>

  <xsl:template match="/">
    <out>
      <xsl:value-of select="app:count-elements()"/>
      <xsl:message>Error not detected!</xsl:message>
    </out>
  </xsl:template>

</xsl:stylesheet>
