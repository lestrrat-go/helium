<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  name="http://www.w3.org/xslt30tests/error-3030a">

                    <!--It is a static error if an xsl:accept element matches no
                        components in the used package, unless the tokens in its names attribute
                        are all wildcards.-->
                        
                    <!-- See also bug 26679 -->
                    
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-002" package-version="1.0.0">
     <xsl:accept component="template" names="non-existent" visibility="public"/>
   </xsl:use-package>                   
                        
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
   
</xsl:package>
