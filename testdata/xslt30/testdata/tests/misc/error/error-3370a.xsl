<?xml version="1.0" encoding="UTF-8"?>
<xsl:package name="http://www.w3.org/tests/error-3370"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!--
      This error does not exist anymore, XTSE0090 should be raised instead.

      The error was XTSE3370:

      It is a static error if the visibility attribute of a stylesheet parameter 
      that specifies required="yes" and does not specify static="yes" is present with a value other 
      than public, final, or abstract, or if an xsl:expose or xsl:accept declaration attempts to modify 
      the visibility of such a component to a value other than public, final, or abstract.
-->
   <xsl:template name="main">
      <out>
         <x p="{$p}"/>
      </out>
   </xsl:template>
   
   <xsl:param name="p" required="yes" visibility="private"/>
</xsl:package>
