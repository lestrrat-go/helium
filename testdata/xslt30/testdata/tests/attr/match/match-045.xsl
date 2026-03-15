<?spec xslt#patterns?>
<!-- Bug 1069950 - patterns of the form ns:* ignore the node kind -->

             <xsl:stylesheet version="2.0"
                  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                  xmlns:xhtml="http://www.w3.org/1999/xhtml">

                 <xsl:template match="xhtml:*">
                     <xsl:element name="{local-name()}">
                         <xsl:apply-templates select="@*|node()"/>
                     </xsl:element>
                 </xsl:template>

                 <xsl:template match="@xhtml:*">
                     <xsl:attribute name="{local-name()}">
                         <xsl:value-of select="."/>
                     </xsl:attribute>
                 </xsl:template>

                 <xsl:template match="@*|node()" priority="-2">
                     <xsl:copy>
                         <xsl:apply-templates select="@*|node()"/>
                     </xsl:copy>
                 </xsl:template>
             </xsl:stylesheet>
