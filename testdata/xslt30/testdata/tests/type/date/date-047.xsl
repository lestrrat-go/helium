<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-year-from-date?>
<?spec fo#func-month-from-date?>
<?spec fo#func-day-from-date?>
<?spec fo#func-timezone-from-date?>
  <!-- PURPOSE: XPath 2.0: test component extraction on date with a timezone. -->

  <xsl:template match="/">
    <xsl:variable name="t" select="xs:date('2002-06-15+05:00')"/>
    <c  y="{year-from-date($t)}"
        M="{month-from-date($t)}"
        d="{day-from-date($t)}"
        tz="{timezone-from-date($t)}"/>
  </xsl:template>

</xsl:stylesheet>