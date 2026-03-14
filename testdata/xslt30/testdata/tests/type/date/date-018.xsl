<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: test whether a specific date is past or future. -->
  <!-- we currently assume that an absent timezone means UTC -->

  <xsl:template match="/">
    <xsl:variable name="now" select="current-dateTime()"/>
    <A>
    <out><xsl:value-of select="xs:dateTime('1996-12-23T13:13:00') &lt;= $now"
                               /></out>
    <out><xsl:value-of select="xs:dateTime('2026-12-23T13:13:00') &lt;= $now"
                               /></out>
    </A>                               
  </xsl:template>

</xsl:stylesheet>