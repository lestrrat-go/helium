<?spec xslt#patterns?>
<!-- Bug 30375 - default priority of Q{uri}* patterns -->

 <xsl:stylesheet version="3.0"
      xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
      xmlns:xhtml="http://www.w3.org/1999/xhtml">

     <xsl:template match="Q{http://www.w3.org/1999/xhtml}*" mode="#all">
         <wildcard>
             <xsl:next-match/>
         </wildcard>
     </xsl:template>
     
     <xsl:template match="xhtml:html" priority="-0.1">
         <minus-0.1>
             <xsl:next-match/>
         </minus-0.1>
     </xsl:template>
     
     <xsl:template match="xhtml:html" priority="-0.4">
         <minus-0.4>
             <xsl:next-match/>
         </minus-0.4>
     </xsl:template>
     
     <xsl:template match="xhtml:html" priority="-2.0">
         <minus-2.0/>
     </xsl:template>
     
     <xsl:template match="xhtml:html" priority="1.5">
         <plus1.5>
             <xsl:next-match/>
         </plus1.5>
     </xsl:template>

     <xsl:template match="/">
         <out>
             <xsl:apply-templates select="//test/*"/>
         </out>
     </xsl:template>

 </xsl:stylesheet>
