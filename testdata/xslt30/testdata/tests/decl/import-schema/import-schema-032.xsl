<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema041.uri/"
        xmlns:xslxx="http://www.w3.org/1999/XSL/Transformxx"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test use of facets controlling string length -->
<?spec xpath#id-castable?>

<xsl:import-schema schema-location="schema041.xsd" 
     namespace="http://schema041.uri/"/>
  



  <xsl:template match="/">
    <out>
      <f><xsl:value-of select="'strict' castable as enu:string10"/></f>
      <t><xsl:value-of select="'straiction' castable as enu:string10"/></t>      
      <f><xsl:value-of select="'strict-rotation' castable as enu:string10"/></f>      
      <f><xsl:value-of select="'laxy' castable as enu:string5-8"/></f>
      <t><xsl:value-of select="'strip' castable as enu:string5-8"/></t>
      <t><xsl:value-of select="'preserve' castable as enu:string5-8"/></t>
      <f><xsl:value-of select="'international' castable as enu:string5-8"/></f>                        
    </out>
  </xsl:template>
  
</xsl:stylesheet>