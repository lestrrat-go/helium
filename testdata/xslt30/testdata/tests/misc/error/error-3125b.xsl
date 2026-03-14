<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                 <!--It is a static error if the
                     select attribute of xsl:break or xsl:on-completion is present and the
                     instruction has children.-->
                     
   <xsl:template name="main">
      <out>
         <xsl:iterate select="1 to 10">
           <xsl:on-completion select="3">
             <xsl:sequence select="29"/>
           </xsl:on-completion>
           <xsl:sequence select="72"/>
         </xsl:iterate>
      </out>
   </xsl:template>
</xsl:stylesheet>
