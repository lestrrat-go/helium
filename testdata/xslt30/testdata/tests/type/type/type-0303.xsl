<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: Test that an error is raised if @type is present 
  				in the stylesheet but the processor is a basic processor.-->

   <xslt:template match="/doc">
      <xslt:element name="elem" type="xs:double">2.2</xslt:element>
   </xslt:template>
</xslt:transform>
