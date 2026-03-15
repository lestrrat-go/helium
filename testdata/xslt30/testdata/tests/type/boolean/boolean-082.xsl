<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:dd="http://dd.com/"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
      <!-- crashes with NPE in Saxon 7.9.1 -->
      <?spec fo#func-compare?>
      <?same-as-1.0 no?>
      <xsl:template match="dd:hcfa-jcodes" as="element()*">
      <zzz>
         <xsl:variable name="non_blank_jcodes">
            <xsl:for-each select="dd:hcfa-jcode">
               <xsl:variable name="nbj" select="normalize-space(.)"/>
               <xsl:if test="string-length($nbj) > 0">
                  <dd:hcfa-jcode><xsl:value-of select="$nbj"/></dd:hcfa-jcode>
               </xsl:if>
            </xsl:for-each>
         </xsl:variable>
            <tr valign="top">
               <td style="color: #336699; font-weight:
bold">HCFA Jcodes:</td>
               <td>
                  <xsl:for-each select="$non_blank_jcodes/dd:hcfa-jcode">
                     <xsl:variable name="this_pos" select="position()"/>
                     <xsl:if
test="compare($non_blank_jcodes/dd:hcfa-jcode[position() &lt; $this_pos]/text(), ./text()) = 0" >
                                            </xsl:if>
                     <xsl:value-of select="." />
                     <xsl:if test="not(position()=last())">, </xsl:if>
                  </xsl:for-each>
               </td>
            </tr>
         </zzz>   
      </xsl:template>


</xsl:stylesheet>
