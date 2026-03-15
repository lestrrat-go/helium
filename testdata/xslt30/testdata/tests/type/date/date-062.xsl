<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs my"
xmlns:my="http://my.uri/">

<?spec fo#date-time-values?>
<?spec fo#func-min?>
<?spec fo#func-max?>
  <!-- PURPOSE: XPath 2.0: test min() and max() on durations -->
  
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
        <xsl:value-of select="min(for $a in $x/a return xs:dayTimeDuration($a))"/>;
        <xsl:value-of select="max(for $a in $x/a return xs:dayTimeDuration($a))"/>;        
    </d>                                       
  </xsl:template>
  


</xsl:stylesheet>