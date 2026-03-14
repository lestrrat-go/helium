<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

                          <!--It is a dynamic error 
                              if an invocation of an absent component (that is, an abstract component accepted
                              into a using package with visibility="absent") is evaluated.-->
                              
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-004" package-version="1.0.0">  
     <xsl:accept component="template" names="t-abstract" visibility="hidden"/>
   </xsl:use-package> 

   <xsl:template name="main">
      <out>
         <xsl:call-template name="t-abstract"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
