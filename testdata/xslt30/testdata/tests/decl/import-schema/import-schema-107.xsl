<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema061.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test validation="strict", including nested validation of child elements -->

<!-- error: missing attribute zip -->

     <?spec xslt#validating-using-validation-attribute?>
<?error ?>

<xsl:import-schema schema-location="schema061.xsd" 
     namespace="http://schema061.uri/"/>
  



  <xsl:template match="/">
    <address xsl:validation="strict">
      <person>
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <age>52</age>
    </address>
  </xsl:template>
  
</xsl:stylesheet>