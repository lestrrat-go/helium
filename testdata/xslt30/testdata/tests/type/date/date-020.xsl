<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
 xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-current-dateTime?>
  <!-- PURPOSE: XPath 2.0: test that current date/time doesn't change
       throughout a transformation -->

  <xsl:template match="/">
    <xsl:variable name="now" select="current-dateTime()"/>
    <!-- waste some time -->
    <xsl:for-each select="1 to 1000000">
    <xsl:if test="position() = -1">zzzzz<a/>zzzzz</xsl:if>
    </xsl:for-each>
    <A>
      <xsl:variable name="then" select="current-dateTime()"/>
      <xsl:value-of select="$then = $now"/>
    </A>                               
  </xsl:template>

</xsl:stylesheet>