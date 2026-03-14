<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
version="2.0">

<!-- Test case reported on xsl-list as a Saxon bug on 9 Mar 2007 -->

<!-- Tests numbering the children of the document node with from="/" -->

<xsl:strip-space elements="*"/>

<xsl:template match="/">
<out><xsl:apply-templates/></out>
</xsl:template>

     <xsl:template match="processing-instruction(thing)">
         <z>
             <from-slash><xsl:number count="processing-instruction('thing')" from="/" level="any"/></from-slash>
             <from-star><xsl:number count="processing-instruction('thing')" from="/*" level="any"></xsl:number></from-star>
         </z>
     </xsl:template>
</xsl:stylesheet>