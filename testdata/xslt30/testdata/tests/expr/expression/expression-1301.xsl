<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs"
>

<!-- Test max() over sequences -->
<?spec fo#func-max?>

<xsl:template match="/">
  <out>;
    <xsl:value-of select="max(())"/>;
    <xsl:value-of select="max((1))"/>;    
    <xsl:value-of select="max((1,2,4))"/>;
    <xsl:value-of select="max(for $x in //item/@val return string($x))"/>;   
    <xsl:value-of select="max(//item/@val)"/>;    
  </out>
</xsl:template>
 
</xsl:stylesheet>
