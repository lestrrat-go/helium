<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:f="http://f.com/">
<!-- Test for an uncaught dynamic error occurring during evaluation of xsl:result-document -->

  
   <xsl:template match="/">
      <out>
         <xsl:analyze-string select="string(.)" regex="(([A-Z])([a-z]+))">
            <xsl:matching-substring>
               <xsl:if test="position() lt 300">
                  <xsl:variable name="pos" select="position()"/>
                  <xsl:message>Writing <xsl:value-of select="concat(., $pos, '.xml')"/></xsl:message>
                  <xsl:result-document href="{.}{$pos}.xml">
                     <out position="{1 to $pos}">
                        <xsl:value-of select="regex-group(3)"/>
                        <xsl:if test="$pos eq 130">
                           <error value="{f:divide(300, $pos - 130)}"/>
                        </xsl:if>
                     </out>
                  </xsl:result-document>
               </xsl:if>
            </xsl:matching-substring>
            <xsl:non-matching-substring/>
         </xsl:analyze-string>
      </out>
   </xsl:template>
   
   <xsl:function name="f:divide">
      <!-- unnecessarily complicated to prevent inlining -->
      <xsl:param name="x"/>
      <xsl:param name="y"/>
      <xsl:iterate select="1 to 1000">
         <xsl:param name="t" select="1"/>
         <xsl:on-completion select="$x div $y"/>
         <xsl:next-iteration>
            <xsl:with-param name="t" select="$t + 21"/>
         </xsl:next-iteration>
      </xsl:iterate>
   </xsl:function>
</xsl:transform>
