<?xml version="1.0" encoding="UTF-8"?>
<!--Where an attribute is defined to contain a pattern, it is a static
                           error if the pattern does not match the production Pattern.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test that PI names may NOT contain a colon (see Namespaces REC section 6) -->
<?spec xslt#errors?><?error XTSE0340?>


   <xsl:template match="*">
      <xsl:copy-of select="."/>
   </xsl:template>
  
   <xsl:template match="comment()">
      <xsl:copy/>
   </xsl:template>

   <xsl:template match="processing-instruction(proc:inst-2)">
      <n>
         <xsl:value-of select="concat(name(), ';', local-name(), ';', namespace-uri())"/>
      </n>
      <xsl:copy/>
   </xsl:template>

</xsl:stylesheet>
