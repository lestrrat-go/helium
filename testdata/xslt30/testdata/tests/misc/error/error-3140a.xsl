<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                    <!--It is a static error if the
                        select attribute of the xsl:try element is
                        present and the element has children other than xsl:catch and
                        xsl:fallback elements.-->
   <xsl:template name="main">
      <out>
         <xsl:try select="1 to 10">
           <xsl:value-of select="19"/>
           <xsl:catch>22</xsl:catch>
         </xsl:try>
      </out>
   </xsl:template>
</xsl:stylesheet>
