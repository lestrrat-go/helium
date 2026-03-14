<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses processing-instruction() node test in the match attribute 
  				of a template and use the content of the processing instructions as the value 
  				of new elements. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="processing-instruction('xml-stylesheet')">
  		  <stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="."/>
      </stylesheet>
	  </t:template>

   <t:template match="processing-instruction('my_window')">
  		  <comp xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="."/>
      </comp>
	  </t:template>

   <t:template match="note">
  		  <p xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:apply-templates/>
      </p>
	  </t:template>

   <t:template match="doc">
  		  <out>
         <t:apply-templates/>
      </out>
	  </t:template>
</t:transform>
