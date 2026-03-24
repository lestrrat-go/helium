<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
 xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-year-from-dateTime?>
<?spec fo#func-month-from-dateTime?>
<?spec fo#func-day-from-dateTime?>
<?spec fo#func-hours-from-dateTime?>
<?spec fo#func-minutes-from-dateTime?>
<?spec fo#func-seconds-from-dateTime?>
<?spec fo#func-timezone-from-dateTime?>
  <!-- PURPOSE: XPath 2.0: test component extraction on dateTime with timezone. -->

  <xsl:template match="/">
    <xsl:variable name="t" select="xs:dateTime('2002-06-15T12:27:30.012+03:00')"/>
    <c  y="{year-from-dateTime($t)}"
        M="{month-from-dateTime($t)}"
        d="{day-from-dateTime($t)}"
        h="{hours-from-dateTime($t)}"
        m="{minutes-from-dateTime($t)}"
        s="{seconds-from-dateTime($t)}"
        tz="{timezone-from-dateTime($t)}"/>
  </xsl:template>

</xsl:stylesheet>