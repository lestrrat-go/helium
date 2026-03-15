<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Launch transformation with initial named template and also with stylesheet 
  				params. Show that the template params are defaulted, but the template can 
  				pick up the stylesheet params through (global) variable references. -->

   <t:param name="Spar" select="'Sdefault'"/>

   <t:template match="/">
 	    <t:comment>ERROR: this value should NOT appear</t:comment>
   </t:template>

   <t:template name="temp">
 	    <t:param name="Tpar" select="'Tdefault'"/>
 	    <out>
         <t:value-of select="concat('stylesheet got: ',$Spar)"/>
         <t:value-of select="concat(', template got: ',$Tpar)"/>
      </out>
   </t:template>
</t:transform>
