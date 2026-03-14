<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
 name="http://www.w3.org/xslt30tests/error-3085a"
 package-version="1.0.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                          <!--It is a static error, unless the value
                              of the declared-modes attribute of an xsl:package
                              element is no, if the mode attribute of
                              an xsl:template or xsl:apply-templates instruction
                              within the package contains, either explicitly or implicitly 
                              (for example, by virtue of a relevant default-mode
                              attribute), a mode name that is not the subject of an xsl:mode declaration
                              appearing within the package nor a public or final xsl:mode
                              declaration accepted from a used package. For this purpose the unnamed mode
                              behaves in the same way as a named mode, so that a reference to the unnamed mode is allowed
                              only if the unnamed mode is declared.-->
                              
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
   
   <xsl:template match="/" mode="undeclared"/>
   
</xsl:package>
