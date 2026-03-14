<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
   name="http://www.w3.org/xslt/package/error-3010a">
   
                    <!--It is a static error if the exposed visibility of a component is inconsistent
                        with its potential visibility, as defined in the above table, unless the token that
                        matches the component is a wildcard, in which case it is treated as not matching that component.-->
                        
   <xsl:template name="main">
      <out>
         <xsl:call-template name="p"/>
      </out>
   </xsl:template>
   
   <xsl:template name="p" visibility="private"/>
   <xsl:expose component="template" names="p" visibility="public"/>
   
</xsl:package>
