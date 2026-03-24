<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                 <!--It is a dynamic error if there are two
                     xsl:merge-key elements that occupy corresponding positions
                     among the xsl:merge-key children of two different
                     xsl:merge-source elements and that have differing effective values for any of the
                     attributes lang, order, collation,
                     case-order, or data-type. Values are considered to differ
                     if the attribute is present on one element and not on the other, or if
                     it is present on both elements with effective
                     values that are not equal to each other. In the case of the
                     collation attribute, the values are compared as absolute URIs
                     after resolving against the base URI.The error may be reported statically if it is detected statically.
                   -->
                   
   <xsl:template name="main">
      <out>
         <xsl:merge>
           <xsl:merge-source name="a" select="1 to 50">
             <xsl:merge-key select="." lang="en"/>
           </xsl:merge-source>
           <xsl:merge-source name="b" select="1 to 50">
             <xsl:merge-key select="." lang="de"/>
           </xsl:merge-source>
           <xsl:merge-action>
             22
           </xsl:merge-action>
         </xsl:merge>
      </out>
   </xsl:template>
</xsl:stylesheet>
