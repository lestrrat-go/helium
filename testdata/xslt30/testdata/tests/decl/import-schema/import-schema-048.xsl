<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema058.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test a complex type involving attributes  -->
              <?spec xslt#import-schema?>
              <?spec xslt#validation-xsl-type?>

<xsl:import-schema schema-location="schema058.xsd" 
     namespace="http://schema058.uri/"/>
  



  <xsl:template match="/">
    <enu:out>
    <z xsl:type="enu:extended" price="23.70" discount="0.01" rebate="0.01"><enu:j/><enu:c/></z>
    <z xsl:type="enu:extended" rebate="0.01"><enu:k/><enu:c/></z>
    <z xsl:type="enu:extended" rebate="0.01"><enu:k/><enu:c/></z>    
    </enu:out>
  </xsl:template>
  
</xsl:stylesheet>