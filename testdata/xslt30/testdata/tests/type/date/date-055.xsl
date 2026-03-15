<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" 
xmlns:xpdt="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xpdt my"
xmlns:my="http://my.uri/">

<?dependency saxon?>
<?spec fo#func-days-from-duration?>
  <!-- PURPOSE: XPath 2.0: test dayTimeDuration. -->
  
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
            <xsl:sort select="xpdt:dayTimeDuration(.)" order="descending" />
            <xsl:variable name="dtd" select="xpdt:dayTimeDuration(text())"/>
            <z><xsl:value-of select="$dtd"/></z>;
            <y><xsl:value-of select="xpdt:dayTimeDuration('PT1S')*(my:as-seconds($dtd))"/></y>
        </xsl:for-each>
    </d>                                       
  </xsl:template>
  
  <xsl:function name="my:as-seconds">
  <xsl:param name="d"/>
  <xsl:sequence select="(((days-from-duration($d) * 24) +
                        hours-from-duration($d)) * 60 +
                        minutes-from-duration($d)) * 60 +
                        seconds-from-duration($d) "/>
  </xsl:function>

</xsl:stylesheet>