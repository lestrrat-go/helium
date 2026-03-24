<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f="f.uri"
xmlns:fn="http://www.w3.org/2005/xpath-functions"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:my="http://my.com/"
exclude-result-prefixes="f fn xs">

<!--  test calling format-number() within xsl:evaluate, with an unknown decimal format -->

<xsl:param name="e1">format-number($p1, '#.00')</xsl:param>
<xsl:param name="e2">format-number($p1, '#,00', 'my:tamrof')</xsl:param>

<xsl:decimal-format name="my:format" decimal-separator="," grouping-separator="'"/>

    
  <xsl:template match="/" name="main">
    <out>
      <xsl:for-each select="1 to 4">
        <z><xsl:evaluate xpath="$e1">
          <xsl:with-param name="p1" select="123.456"/>
          </xsl:evaluate>
        </z>
        <z><xsl:evaluate xpath="$e2">
          <xsl:with-param name="p1" select="456.789"/>
          </xsl:evaluate>
        </z>
        
      </xsl:for-each>  
    </out>
      
  </xsl:template>
  
 
</xsl:stylesheet>
