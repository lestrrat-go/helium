<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Basic test of xsl:iterate; checks that xsl:fallback is permitted -->

<xsl:template match="/">
<out>
  <xsl:iterate select="//ITEM/TITLE" >
    <item position="{position()}" last="{last()}"><xsl:copy-of select="."/></item>
    <xsl:fallback>
      <xsl:for-each select="//ITEM/TITLE">
        <item position="{position()}" last="{last()}"><xsl:copy-of select="."/></item>
      </xsl:for-each>
    </xsl:fallback>  
  </xsl:iterate>
</out>
</xsl:template>

</xsl:stylesheet>