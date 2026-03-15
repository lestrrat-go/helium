<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs"
>

<!-- Test min() over sequences -->
<?spec fo#func-min?>

<xsl:template match="/">
  <out>;
    <xsl:value-of select="min(())"/>;
    <xsl:value-of select="min((1))"/>;    
    <xsl:value-of select="min((-1,2,4))"/>;
    <xsl:value-of select="min(for $x in //item/@val return xs:string($x))"/>; 
    <xsl:value-of select="min(//item/@val)"/>;     
  </out>
</xsl:template>
 
</xsl:stylesheet>
