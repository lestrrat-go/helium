<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                    <!--It is a static error if the
                        select attribute of the xsl:catch element is
                        present unless the element has empty content.-->
   <xsl:template name="main">
      <out>
         <xsl:try select="1 to 10">
           <xsl:catch select="22">
             <xsl:value-of select="23"/>
           </xsl:catch>
         </xsl:try>
      </out>
   </xsl:template>
</xsl:stylesheet>
