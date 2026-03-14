<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
				xmlns="http://www.w3.org/1999/xhtml" 
				version="2.0">

  <!-- Purpose: Test with unsupported value for @encoding, output method is XHTML. -->
  
  <!-- Serialization 5.1.2: A serialization error [err:SESU0007] occurs if an output encoding 
    other than UTF-8 or UTF-16 is requested and the serializer does not support that encoding. 
    The serializer MUST signal the error, or recover by using UTF-8 or UTF-16 instead. --> 

<xsl:output method="xhtml" encoding="XXX-xx" indent="no" />

<xsl:template match="/">
	<doc>hello</doc>
</xsl:template>

</xsl:stylesheet>
