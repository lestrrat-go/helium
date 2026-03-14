<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-years-from-duration?>
<?spec fo#func-months-from-duration?>

  <!-- PURPOSE: XPath 2.0: test yearMonthDuration. -->
  
  <xsl:variable name="x">
  <a>P1Y</a>
  <a>P11M</a>
  <a>P1Y6M</a>
  <a>P1M</a>
  <a>P17M</a>
  </xsl:variable>

  <xsl:template match="/">
    <d>
        <xsl:for-each select="$x/a">
            <xsl:sort select="xs:yearMonthDuration(.)" />
            <xsl:variable name="ym" select="xs:yearMonthDuration(text())"/>
            <z><xsl:value-of select="$ym"/></z>
            <y><xsl:value-of select="xs:yearMonthDuration('P1M') * (years-from-duration($ym)*12 + months-from-duration($ym))"/></y>
        </xsl:for-each>
    </d>                                       
  </xsl:template>

</xsl:stylesheet>