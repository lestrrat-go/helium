<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema068.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test validation of a final result tree -->
<!-- ERROR: non-unique id value. This becomes an error as a consequence of erratum E12.
     Note that the requirement to check uniqueness constraints is only a "should", not a "must",
     to preserve compatibility with the first edition XSLT 2.0 specification -->
<?spec xslt#validating-document-nodes?>
<?error ?>

<xsl:import-schema schema-location="schema068.xsd" 
     namespace="http://schema068.uri/"/>
  



  <xsl:template match="/" name="main">
    <out xsl:validation="strict">
      <person id="p1">
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <person id="p1">
        <first>Penny</first>
        <surname>Whelan</surname>
      </person>
      <person id="p3">
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