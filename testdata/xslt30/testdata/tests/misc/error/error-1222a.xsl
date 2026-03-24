<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                   <!--It is a static error if there are several xsl:key
                        declarations in the stylesheet with
                        the same key name and different effective values for the composite
                        attribute.-->
                        
   <xsl:key name="k" match="a" use="b" composite="yes"/> 
   <xsl:key name="k" match="p" use="q" composite="no"/>                     
                        
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
