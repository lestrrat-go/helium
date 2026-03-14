<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs my"
xmlns:my="http://my.uri/">

<?spec fo#func-multiply-dayTimeDuration?>
<?spec fo#func-divide-dayTimeDuration?>
  <!-- PURPOSE: XPath 2.0: test multiplying and dividing dayTimeDuration by a number -->
  
  <xsl:variable name="x">
  <a>P2D</a>
  <a>P1DT36H</a>
  <a>P1DT6H</a>
  <a>PT1H0M0S</a>
  <a>PT3600S</a>
  <a>PT10.03S</a> 
  <a>PT10.02S</a> 
  <a>PT0S</a> 
  <a>-PT100S</a>   
  </xsl:variable>

  <xsl:template match="/">
    <d>
        <xsl:for-each select="$x/a">
        <e>
            <f><xsl:value-of select="xs:dayTimeDuration(.) * 3"/></f>;
            <f><xsl:value-of select="0.5 * xs:dayTimeDuration(.)"/></f>;            
            <f><xsl:value-of select="xs:dayTimeDuration(.) div 2"/></f>;   
        </e>;
        </xsl:for-each>
    </d>                                       
  </xsl:template>
  


</xsl:stylesheet>