<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  
version="2.0">

     <xsl:template match="/">
     <out>
         <a><xsl:value-of select="//comment()[1]"/></a>
         <b><xsl:value-of select="count(//node())"/></b>
         <c><xsl:value-of select="/comment()/following-sibling::node()[1]/name()"/></c>
     </out>    
     </xsl:template>

</xsl:stylesheet>