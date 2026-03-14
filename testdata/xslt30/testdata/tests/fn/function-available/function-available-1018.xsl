<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="3.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 exclude-result-prefixes="xs"
>

<!-- Test function-available on higher-order functions -->

  <xsl:param name="hof" select="system-property('xsl:supports-higher-order-functions') eq 'yes'"/>
  <xsl:param name="xp31" select="xs:decimal(system-property('xsl:xpath-version')) ge 3.1"/>
  
  <xsl:template match="/">
    <out hof="{$hof}" xp31="{$xp31}">
      <a><xsl:value-of select="$hof eq function-available('function-name', 1)"/></a> 
      <b><xsl:value-of select="$hof eq function-available('function-arity', 1)"/></b>   
      <c><xsl:value-of select="$hof eq function-available('filter', 2)"/></c> 
      <d><xsl:value-of select="$hof eq function-available('for-each', 2)"/></d> 
      <e><xsl:value-of select="$hof eq function-available('function-lookup', 2)"/></e>   
      <f><xsl:value-of select="$hof eq function-available('fold-left', 3)"/></f> 
      <g><xsl:value-of select="$hof eq function-available('fold-right', 3)"/></g>
      <h><xsl:value-of select="$hof eq function-available('for-each-pair', 3)"/></h>
      <i><xsl:value-of select="($hof and $xp31) eq function-available('sort', 3)"/></i>
      <j><xsl:value-of select="($hof and $xp31) eq function-available('apply', 2)"/></j>
    </out>
  </xsl:template>
  

    
</xsl:stylesheet>