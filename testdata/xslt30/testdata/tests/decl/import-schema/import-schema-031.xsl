<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema040.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test use of a used-defined enumeration type -->
<?spec xpath#id-castable?>

<xsl:import-schema schema-location="schema040.xsd" 
     namespace="http://schema040.uri/"/>


  <xsl:template match="/">
    <out>
      <t><xsl:value-of select="'strict' castable as enu:validation"/></t>
      <t><xsl:value-of select="'lax' castable as enu:validation"/></t>
      <t><xsl:value-of select="'strip' castable as enu:validation"/></t>
      <t><xsl:value-of select="'preserve' castable as enu:validation"/></t>
      <f><xsl:value-of select="'banana' castable as enu:validation"/></f>                        
    </out>
  </xsl:template>
  
</xsl:stylesheet>