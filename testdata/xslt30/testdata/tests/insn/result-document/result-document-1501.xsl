<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Test for a potential bug when xsl:result-document is multi-threaded:
   us xsl:result-document within xsl:matching-substring of xsl:analyze-string -->

   
   <xsl:template match="/">
      <out>
         <xsl:analyze-string select="string(.)" regex="(([A-Z])([a-z]+))">
            <xsl:matching-substring>
               <xsl:if test="position() lt 300">
                  <xsl:message>Writing <xsl:value-of select="concat(., position(), '.xml')"/></xsl:message>
                  <xsl:result-document href="{.}{position()}.xml">
                     <out position="{1 to position()}">
                        <xsl:value-of select="regex-group(3)"/>
                     </out>
                  </xsl:result-document>
               </xsl:if>
            </xsl:matching-substring>
            <xsl:non-matching-substring/>
         </xsl:analyze-string>
      </out>
   </xsl:template>
</xsl:transform>
