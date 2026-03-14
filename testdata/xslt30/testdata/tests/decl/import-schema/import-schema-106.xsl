<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema060.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test a complex type with empty content model  -->

     <?spec xslt#validation-xsl-type?>
<?error ?>

<xsl:import-schema schema-location="schema060.xsd" 
     namespace="http://schema060.uri/"/>
  



  <xsl:template match="/">
    <enu:out>
    <z xsl:type="enu:sc" cost="23.45">fred</z>
    </enu:out>
  </xsl:template>
  
</xsl:stylesheet>