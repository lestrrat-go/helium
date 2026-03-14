<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!--[ERR XTSE3051] It is a static error if a token in the names attribute of xsl:accept, 
    other than a wildcard, matches the symbolic name of a component declared within 
    an xsl:override child of the same xsl:use-package element.-->

   
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-002" package-version="1.0.0">
     <xsl:override>
       <xsl:template name="t-public" as="xs:string">t-public-override</xsl:template>
     </xsl:override>
     <xsl:accept component="template" names="t-final t-public" visibility="final"/>
     
   </xsl:use-package> 
   
 
   <xsl:template name="main">
      <out>
         <xsl:call-template name="t-public"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
