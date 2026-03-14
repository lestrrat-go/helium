<?spec xslt#defining-templates?>
<!-- Check that a template rule with negative priority is OK -->

             <xsl:stylesheet version="2.0"
                  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

                 <xsl:template match="/">
                     <d><xsl:apply-templates/></d>
                 </xsl:template>

                 <xsl:template match="*" priority="-17">
                     <ok/>
                 </xsl:template>
             </xsl:stylesheet>
