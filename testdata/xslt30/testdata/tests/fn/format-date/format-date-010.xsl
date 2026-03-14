<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: ISO day within week -->
  <!-- Specifically, in the ISO calendar the days of the week are numbered from
        1 (Monday) to 7 (Sunday), and week 1 in any calendar year is the week
         (from Monday to Sunday) that includes the first Thursday of that year.-->

  <xsl:param name="t" as="xs:date" select="xs:date('2003-12-01')"/>  

  <xsl:template name="main">
<out>;
<xsl:for-each select="1 to 48">
<xsl:variable name="d" select="$t + xs:yearMonthDuration('P1M')*."/>
<t date="{$d}"><xsl:value-of select="format-date($d, '[F01]', (), 'ISO', ())"/></t>;
</xsl:for-each>
</out>     
  </xsl:template>

</xsl:stylesheet>