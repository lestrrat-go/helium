<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
                 <!--It is a static error if an
                     xsl:break or xsl:next-iteration element
                     appears other than in a tail position within the sequence
                     constructor forming the body of an xsl:iterate instruction.-->
   <xsl:template name="main">
      <out>
         <xsl:iterate select="1 to 50">
           <xsl:if test=". gt 30"><xsl:break/></xsl:if>
           <x/>
         </xsl:iterate>
      </out>
   </xsl:template>
</xsl:stylesheet>
