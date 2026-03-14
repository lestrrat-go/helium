<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                    <!--It is a static error if the visibility assigned to a component
                        by an xsl:accept element is incompatible with the
                        visibility of the corresponding component in the used package, as defined
                        by the above table, unless the token that matches the component name is a wildcard,
                        in which case the xsl:accept element is treated as not matching
                        that component.-->
                        
   <xsl:use-package name="http://www.w3.org/xslt30tests/error-base-002" package-version="1.0.0">
     <xsl:accept component="template" names="t-private" visibility="public"/>
   </xsl:use-package> 
   

   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
