<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-hours-from-time?>
<?spec fo#func-minutes-from-time?>
<?spec fo#func-seconds-from-time?>
<?spec fo#func-timezone-from-time?>

  <!-- PURPOSE: XPath 2.0: test component extraction on time with timezone. -->

  <xsl:template match="/">
    <xsl:variable name="t" select="xs:time('12:27:30.012+03:00')"/>
    <c  
        h="{hours-from-time($t)}"
        m="{minutes-from-time($t)}"
        s="{seconds-from-time($t)}"
        tz="{timezone-from-time($t)}"/>
  </xsl:template>

</xsl:stylesheet>