<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <xsl:template name="main">
   
                <xsl:merge-source select="doc('log-file-1.xml')">
                      
                        
                        <xsl:merge-key select="dateTime(current-date(), @time)" />
                           
                        
                   
                </xsl:merge-source>
             
            
        
    
    </xsl:template>
</xsl:stylesheet>