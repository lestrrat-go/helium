<?xml version='1.0' encoding="UTF-8"?>
<x:stylesheet xmlns:x="http://www.w3.org/1999/XSL/Transform"
              xmlns:xsl="http://anything/" version="2.0">

<x:namespace-alias stylesheet-prefix="xsl" result-prefix="x"/>

<!-- BUG: not actually a bug -->
<!-- DESCRIPTION: tests xsl:namespace-alias -->

<!-- everything between the x:template tags 
     is outputted to the new stylesheet 
  -->
  
   <x:template match="/">
      <xsl:stylesheet version="1.0">
      <xsl:template match="/">
         <xsl:value-of select="foobar"/>
      </xsl:template>

      <!-- Create the include lines??? -->
      <x:for-each select="//INCLUDE">
         <xsl:include>
            <x:attribute name="href">
               <x:value-of select="."/>
            </x:attribute> 
         </xsl:include>
      </x:for-each>

      </xsl:stylesheet>
   </x:template>
</x:stylesheet>

