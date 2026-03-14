<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema046.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test use type attribute on literal result element, with a simple type -->
              <?spec xslt#import-schema?>
              <?spec xslt#setting-annotation-for-lre?>

<xsl:import-schema schema-location="schema046.xsd" 
     namespace="http://schema046.uri/"/>
  



  <xsl:template match="/">
    <out>
        <a xsl:type="enu:standardMethod">xml</a>
        <a xsl:type="enu:standardMethod">html</a>
        <a xsl:type="enu:standardMethod">text</a>                              
    </out>
  </xsl:template>
  
</xsl:stylesheet>