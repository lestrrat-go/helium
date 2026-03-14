<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                    <!--It is a static error if the
                        name attribute of an xsl:with-param child of
                        an xsl:next-iteration element does not match the
                        name attribute of an xsl:param child of the
                        innermost containing
                        xsl:iterate instruction.-->
   <xsl:template name="main">
      <out>
         <xsl:iterate select="1 to 50">
           <xsl:param name="p" select="2"/>
           <xsl:iterate select="1 to 10">
             <xsl:next-iteration>
                <xsl:with-param name="p" select="5"/>
             </xsl:next-iteration>
           </xsl:iterate>
         </xsl:iterate>
      </out>
   </xsl:template>
</xsl:stylesheet>
