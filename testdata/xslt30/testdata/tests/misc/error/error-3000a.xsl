<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  name="http://www.w3.org/xslt30tests/error-3000a"
  package-version="1.0.0">
  
              <!--It is a static error if no package matching the package name and version specified
                  in an xsl:use-package declaration can be located.-->
                  
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-002" package-version="2.0.0">
     <xsl:accept component="template" names="t-public" visibility="public"/>
   </xsl:use-package>                
                  
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:package>
