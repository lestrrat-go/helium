<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs my"
xmlns:my="http://my.uri/">

<?spec fo#func-multiply-yearMonthDuration?>
<?spec fo#func-divide-yearMonthDuration?>
  <!-- PURPOSE: XPath 2.0: test multiplying and dividing yearMonthDuration by a number -->
  
  <xsl:variable name="x">
  <a>P1Y</a>
  <a>P2Y</a>
  <a>P1Y6M</a>
  <a>P18M</a>
  <a>-P1Y</a>
  <a>-P1M</a> 
  </xsl:variable>

  <xsl:template match="/">
    <d>
        <xsl:for-each select="$x/a">
        <e>
            <f><xsl:value-of select="xs:yearMonthDuration(.) * 3"/></f>;
            <f><xsl:value-of select="0.5 * xs:yearMonthDuration(.)"/></f>;            
            <f><xsl:value-of select="xs:yearMonthDuration(.) div 2"/></f>;   
        </e>;
        </xsl:for-each>
    </d>                                       
  </xsl:template>
  


</xsl:stylesheet>