<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
  <xsl:import-schema schema-location="../docs/loans.xsd"/>  
   
  <!-- within xsl:source-document, use xsl:fork: 2 prongs, both consuming -->
  
  <xsl:template name="f-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:sequence>
            <a><xsl:value-of select="round(avg(/BOOKLIST/BOOKS/ITEM/PRICE), 2)"/></a>
          </xsl:sequence>
          <xsl:sequence>
            <b><xsl:value-of select="round(avg(/BOOKLIST/BOOKS/ITEM/PAGES), 2)"/></b>
          </xsl:sequence>
        </xsl:fork>   
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:fork with some motionless prongs -->
  
  <xsl:template name="f-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:sequence select="'['"/>
          <xsl:sequence>
            <a><xsl:value-of select="round(avg(/BOOKLIST/BOOKS/ITEM/PRICE), 2)"/></a>
          </xsl:sequence>
          <xsl:sequence>
            <b><xsl:value-of select="round(avg(/BOOKLIST/BOOKS/ITEM/PAGES), 2)"/></b>
          </xsl:sequence>
          <xsl:sequence select="']'"/>
        </xsl:fork>   
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:fork, splitting the input to multiple result files (the classic "coloured widget" use case) -->
  
  <xsl:template name="f-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:fork>
          <xsl:sequence>
            <xsl:result-document href="debits.xml">
              <debits>
                <xsl:copy-of select="/*/transaction[number(@value) lt 0]"/>
              </debits>
            </xsl:result-document>  
          </xsl:sequence>
          <xsl:sequence>
            <xsl:result-document href="credits.xml">
              <credits>
                <xsl:copy-of select="/*/transaction[number(@value) ge 0]"/>
              </credits>
            </xsl:result-document>  
          </xsl:sequence>
        </xsl:fork>   
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:fork with all prongs motionless -->
  
  <xsl:template name="f-004" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <in>
          <xsl:fork>
            <xsl:sequence select="'['"/>
            <xsl:sequence>
              <a/>
            </xsl:sequence>
            <xsl:sequence>
              <b/>
            </xsl:sequence>
            <xsl:sequence select="']'"/>
          </xsl:fork> 
        </in>  
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- xsl:source-document / xsl:for-each / xsl:fork -->
  
  <xsl:template name="f-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="v" as="xs:anyAtomicType+">
            <xsl:fork>
              <xsl:sequence select="string(TITLE)"/>
              <xsl:sequence select="number(PAGES)"/>
              <xsl:sequence select="number(PRICE)"/>
              <xsl:sequence select="name(.)"/>
            </xsl:fork>
          </xsl:variable>
          <book title="{$v[1]}" price-per-page="{round($v[3] div $v[2], 4)}" element="{$v[4]}"/>  
        </xsl:for-each>
      </xsl:source-document>  
    </out>
  </xsl:template>
  
  <!-- xsl:fork can return streamed nodes if only one branch is consuming -->
  
  <xsl:template name="f-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
          <book>
            <xsl:fork>
              <xsl:sequence>
                <xsl:attribute name="category" select="@CAT"/>
              </xsl:sequence>
              <xsl:sequence select="TITLE"/>
            </xsl:fork>
          </book>      
        </xsl:for-each>
      </xsl:source-document>  
    </out>
  </xsl:template>
  
  <!-- xsl:fork with try/catch within a single prong -->
  
  <xsl:template name="f-007" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
            <xsl:fork>
              <xsl:sequence select="sum(outermost(//PAGES))"/>
              <xsl:sequence>
                <xsl:try>
                  <xsl:sequence select="string(outermost(//TITLE))"/>
                  <xsl:catch errors="*:XPTY0004"/>
                </xsl:try>
              </xsl:sequence>    
              <xsl:sequence select="round(avg(outermost(//PRICE)), 2)"/>
            </xsl:fork>
      </xsl:source-document>  
    </out>
  </xsl:template>
  
  <!-- xsl:fork with try/catch around the whole xsl:fork -->
  
  <xsl:template name="f-008" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:try>
            <xsl:fork>
              <xsl:sequence select="sum(outermost(//PAGES))"/>
              <xsl:sequence select="string(outermost(//TITLE))"/>   
              <xsl:sequence select="round(avg(outermost(//PRICE)), 2)"/>
            </xsl:fork>
           <xsl:catch errors="*:XPTY0004"/>
         </xsl:try>
      </xsl:source-document>  
    </out>
  </xsl:template>
  
  <!-- xsl:fork used to create attribute nodes -->
  
  <xsl:template name="f-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
          <book>
            <xsl:fork>
              <xsl:sequence>
                <xsl:attribute name="title" select="TITLE"/>
              </xsl:sequence>
              <xsl:sequence>
                <xsl:attribute name="author" select="AUTHOR"/>
              </xsl:sequence>
              <xsl:sequence>
                <xsl:attribute name="publisher" select="PUBLISHER"/>
              </xsl:sequence>
              <xsl:sequence>
                <xsl:attribute name="price" select="PRICE"/>
              </xsl:sequence>              
            </xsl:fork>
          </book>      
        </xsl:for-each>
      </xsl:source-document>  
    </out>
  </xsl:template>
  
  

  
</xsl:transform>  