<?spec xslt#defining-templates?>
<!-- Check that child::document-node() is legal
     (though it never matches anything)  -->

             <xsl:stylesheet version="2.0"
                  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

                 <xsl:template match="child::document-node()">
                     <d>WRONG</d>
                 </xsl:template>
                 
                 <xsl:template match="*">
                     <out>OK</out>
                 </xsl:template>

             </xsl:stylesheet>
