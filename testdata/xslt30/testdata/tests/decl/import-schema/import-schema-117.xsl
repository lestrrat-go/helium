<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema068.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test use xs:all validation -->

<xsl:import-schema schema-location="schema068.xsd" 
     namespace="http://schema068.uri/"/>
     <?spec xslt#validating-using-validation-attribute?>
<?error ?>
  



  <xsl:template match="/">
    <out>
      <person xsl:validation="strict">
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <person xsl:validation="strict">
        <first>Michael</first>
        <surname>Kay</surname>
      </person>
      <person xsl:validation="strict">
        <last>Kay</last>
        <first>Michael</first>
        <middle>H</middle>
      </person> 
      <person xsl:validation="strict">
        <last>Kay</last>
        <first>Michael</first>
      </person>                
    </out>
  </xsl:template>
  
</xsl:stylesheet>