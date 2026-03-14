<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema055.uri/"
        xmlns:carrot="http://carrot.uri/"
        xmlns:banana="http://banana.uri/"
        exclude-result-prefixes="xs enu carrot banana">

<!-- PURPOSE: Test a complex type involving an ##target wildcard  -->

    <?spec xslt#validation-xsl-type?>
     <?error ?>

<xsl:import-schema schema-location="schema055.xsd" 
     namespace="http://schema055.uri/"/>
  



  <xsl:template match="/">
    <enu:out>
    <z xsl:type="enu:repeated-choice"><enu:p/><enu:v/><enu:x/><enu:m/></z>
    <z xsl:type="enu:repeated-choice"><enu:p/><enu:p/><enu:q/><banana:strange/><enu:p/><enu:r/><enu:q/></z>
    </enu:out>
  </xsl:template>
  
</xsl:stylesheet>