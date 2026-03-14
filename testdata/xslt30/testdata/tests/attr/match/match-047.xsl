<?spec xslt#defining-templates?>
<!-- Check that a template rule using the attribute axis not in the last step is legal
     (though it never matches anything)  -->

             <xsl:stylesheet version="2.0"
                  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

                 <xsl:template match="/">
                     <d><xsl:apply-templates/></d>
                 </xsl:template>
                 
                 <xsl:template match="text()"/>

                 <xsl:template match="@a//@a/@a">
                     <should-not-appear/>
                 </xsl:template>
             </xsl:stylesheet>
